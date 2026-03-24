<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" version="3.0">
    <xsl:template name="xsl:initial-template">
        <out>
            <!-- error: key in map should be a QName, not a string -->
            <xsl:variable name="map" select="map { 'beast' : xs:integer(666) }"/>
            <xsl:variable name="xpath2" select="'$beast + 1'"/>
            <xsl:evaluate xpath="$xpath2" with-params="$map"/>
        </out>
    </xsl:template>
</xsl:stylesheet>