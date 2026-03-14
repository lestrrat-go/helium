<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <xsl:param name="propname" as="xs:string" select="'version'"/>
    
    <xsl:template name="xsl:initial-template">
        <xsl:variable name="props" select="available-system-properties()" as="xs:QName+" />
        <xsl:variable name="propQName" select="QName('http://www.w3.org/1999/XSL/Transform', $propname)" />
        <result>
            <xsl:value-of select="system-property('xsl:' || $propname) eq $props[. eq $propQName] ! system-property('xsl:' || local-name-from-QName(.))"/>
        </result>
    </xsl:template>
    
</xsl:stylesheet>