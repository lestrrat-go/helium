<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    xmlns:f="http://local/"
    version="3.0"
    expand-text="yes">
    
    <!-- function-lookup with a context-sensitive system function -->
    
    <xsl:variable name="words" as="element()*">
        <word xml:lang="en">horse</word>
        <word xml:lang="de">pferd</word>
        <word xml:lang="fr">cheval</word>
    </xsl:variable>
       
    <xsl:template name="xsl:initial-template">
        <xsl:variable name="testers" as="map(*)">
            <xsl:map>
                <xsl:for-each select="$words">
                    <xsl:map-entry key="string(.)" select="function-lookup(QName('http://www.w3.org/2005/xpath-functions', 'lang'), 1)"/>
                </xsl:for-each>
            </xsl:map>
        </xsl:variable>
        <out>
            <xsl:value-of select="$words[$testers(.)('de')]"/>
        </out>
    </xsl:template>


</xsl:stylesheet>