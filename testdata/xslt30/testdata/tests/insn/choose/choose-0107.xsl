<xsl:stylesheet 
    xmlns:xsl    = "http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs    = "http://www.w3.org/2001/XMLSchema" 
    xmlns:svg    = "http://www.w3.org/2000/svg" 
    exclude-result-prefixes="xs" 
    xmlns:w="http://www.werum.com" 
    version="2.0">
    
    <!-- A nasty little test in which the same comparison is done repeatedly using different default collations -->
    
    <xsl:param name="u" select="'GREEN23'" as="xs:string*"/>
    <xsl:param name="q" select="'green'" as="xs:string*"/>
    
    <xsl:template name="main">
        <out>            
            <xsl:choose>
                <xsl:when test="$u = concat($q, '23')">One</xsl:when>
                <xsl:when test="$u = concat($q, '23')" default-collation="http://www.w3.org/2013/collation/UCA?strength=secondary">Two</xsl:when>
                <xsl:otherwise>Fail</xsl:otherwise>               
            </xsl:choose>
        </out>
    </xsl:template>        
</xsl:stylesheet>