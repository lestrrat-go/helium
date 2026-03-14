<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:strip-space elements="*"/>
    
    <xsl:mode on-no-match="shallow-copy"/>
    
    <xsl:key
        name="unique-name-id-method" 
        composite="yes" 
        match="*[@id and @method]" 
        use="node-name(), @id, @method"/>
    
    <xsl:template match="*[@id and @method and not(. is key('unique-name-id-method', (node-name(), @id, @method), ..)[1])]"/>
    
</xsl:stylesheet>