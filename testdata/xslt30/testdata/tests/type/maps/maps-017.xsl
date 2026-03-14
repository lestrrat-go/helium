<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xpath-default-namespace="http://www.example.com">
    
    <xsl:output method="json"/>
    
    <xsl:template match="/">
        <xsl:variable name="v" as="map(*)">
            <xsl:map-entry key="string(//bar)" select="'greeting'"/>
        </xsl:variable>
        <out>
            <xsl:value-of select="serialize($v, map{'method':'json'})"/>
        </out>
    </xsl:template>
    
</xsl:stylesheet>