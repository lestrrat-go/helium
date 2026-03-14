<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="#all"
    >
    
    <xsl:mode streamable="yes"/>
    
    <xsl:template match="/">
        <data>
            <xsl:for-each select="/*/part-list/score-part">
                <xsl:variable name="partId" select="string(@id)"/>
                <xsl:variable name="partName" select="string(part-name[1])"/>
                <xsl:source-document href="Psalm100.xml" streamable="yes">
                    <part id="{$partId}" name="{$partName}" measures="{count(//measure[parent::part/@id=$partId])}"/>	
                </xsl:source-document>
            </xsl:for-each>
        </data>     
    </xsl:template>

</xsl:stylesheet>