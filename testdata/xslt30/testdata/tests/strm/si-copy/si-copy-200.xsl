<xsl:stylesheet
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    exclude-result-prefixes="xs" 
    version="3.0">
    
    <xsl:param name="search" as="xs:string" select="'ghi'"/>
    <xsl:param name="replace" as="xs:string" select="'foo'"/>
    
    <xsl:mode streamable="yes" on-no-match="shallow-copy"/>
    
    <xsl:mode name="change" on-no-match="shallow-copy"/>
    
    <xsl:template match="Address">
        <xsl:apply-templates select="copy-of()" mode="change"/>
    </xsl:template>
    
    <xsl:template match="Address[LastName = $search]/FirstName" mode="change">
        <xsl:copy>
            <xsl:value-of select="$replace"/>
        </xsl:copy>
    </xsl:template>
    
</xsl:stylesheet>