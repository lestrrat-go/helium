<xsl:stylesheet 
    xmlns:xsl    = "http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs    = "http://www.w3.org/2001/XMLSchema" 
    xmlns:svg    = "http://www.w3.org/2000/svg" 
    exclude-result-prefixes="xs" 
    xmlns:w="http://www.werum.com" 
    version="2.0">
    
    <!-- A nasty little test in which the same comparison is done repeatedly using different namespace
        contexts -->
    
    <xsl:param name="u" select="xs:untypedAtomic('my:problem')"/>
    <xsl:param name="q" select="QName('http://uri.three/', 'problem')"/>
    
    <xsl:template name="main">
        <out>            
            <xsl:choose>
                <xsl:when test="$u = $q" xmlns:my="http://uri.one/">One</xsl:when>
                <xsl:when test="$u = $q" xmlns:my="http://uri.two/">Two</xsl:when>
                <xsl:when test="$u = $q" xmlns:my="http://uri.three/">Three</xsl:when>
                <xsl:otherwise>Fail</xsl:otherwise>               
            </xsl:choose>
        </out>
    </xsl:template>        
</xsl:stylesheet>