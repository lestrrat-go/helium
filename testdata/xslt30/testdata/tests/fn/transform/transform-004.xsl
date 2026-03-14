<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0"
    expand-text="yes">
    
    <xsl:variable name="decider" static="yes" select="transform( map { 'stylesheet-location': 'transform-004a.xsl', 
        'initial-template' : QName('', 'get-function'), 'delivery-format' : 'raw'})?output"/>
    
    <xsl:template name="main">
        <out xsl:use-when="$decider(-1)">42</out>
        <out xsl:use-when="not($decider(-1))">24</out>
    </xsl:template>

</xsl:stylesheet>