<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="#all"
    version="3.0">
    
    <xsl:character-map name="escape-apos">
        <xsl:output-character character="'" string="&amp;apos;"/>
    </xsl:character-map>
    
    <xsl:output method="adaptive" indent="yes" use-character-maps="escape-apos"/>
    
    <xsl:template match="/" name="xsl:initial-template">
        <test>\'E4</test>
        <xsl:sequence select="'\''E4'"/>
        <xsl:sequence select="map { 'ä' : '\''E4' }"/>
    </xsl:template>
    
</xsl:stylesheet>