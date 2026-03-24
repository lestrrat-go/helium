<xsl:package 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    name="http://www.oup.com/testParamOverride" 
    package-version="0.1" 
    exclude-result-prefixes="#all" 
    version="3.0">
    
    <xsl:mode name="transform" visibility="public"/>
    
    <xsl:param name="newRootElement" as="xs:string" required="no" select="'null'"/>
    
    <xsl:template match="/" mode="transform">
        <xsl:message>New root element: <xsl:value-of select="$newRootElement"/></xsl:message>
        <xsl:element name="{$newRootElement}">
            <xsl:apply-templates mode="#current"/>
        </xsl:element>
    </xsl:template>
    
    <xsl:template match="*" mode="transform">
        <xsl:copy>
            <xsl:apply-templates select="@*,child::node()" mode="#current"/>
        </xsl:copy>
    </xsl:template>
    
    <xsl:template match="@* | comment() | processing-instruction() | text()" mode="transform">
        <xsl:copy-of select="."/>
    </xsl:template>
    
</xsl:package>