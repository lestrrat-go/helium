<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- Two files in a single merge-source -->

    <xsl:template name="main">
        
        <prices>
            <xsl:merge>
                <xsl:merge-source
                    streamable="yes"
                    for-each-source="('../docs/goldPrice.xml', '../docs/uraniumPrice.xml')" 
                	select="dataset/data/datum">
                        <xsl:merge-key select="xs:date(datum[1])" order="descending"/>
                </xsl:merge-source>
                <xsl:merge-action>
                    <key date="{current-merge-key()}" count="{count(current-merge-group())}"/>
                </xsl:merge-action>
            </xsl:merge>
        </prices>
        
    </xsl:template>
</xsl:stylesheet>