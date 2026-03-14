<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:fn="http://www.w3.org/2005/xpath-functions"
    expand-text="yes"
    exclude-result-prefixes="#all"
    version="3.0">
    
    <!-- function conversion rules invoked for xsl:with-param -->
    
    <xsl:output method="xml"/>
    
    <xsl:variable name="in">
        <tome name="pentateuch">
            <book len="82">Genesis</book>
            <book len="84.3">Exodus</book>
            <book len="29.6">Leviticus</book>
            <book len="1e0">Numbers</book>
            <book len="22">Deuteronomy</book>
        </tome>
    </xsl:variable>
    
    <xsl:template name="xsl:initial-template">
        <out>
            <xsl:apply-templates select="$in/tome"/>
        </out>
    </xsl:template>

    <xsl:template match="tome">       
        <xsl:iterate select="*">
            <xsl:param name="total" select="0"/>
            <xsl:on-completion>
                <xsl:sequence select="$total instance of xs:double*"/>
            </xsl:on-completion>
            <xsl:next-iteration>
                <xsl:with-param name="total" select="$total, @len" as="xs:double*"/>
            </xsl:next-iteration>
        </xsl:iterate>  
    </xsl:template>
    
 
    
</xsl:stylesheet>