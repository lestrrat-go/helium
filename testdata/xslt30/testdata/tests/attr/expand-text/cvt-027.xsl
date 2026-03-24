<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

    <xsl:variable name="Q{http://www.example.com/foo}var" select="'London'"/>

    <xsl:template name="main" expand-text="yes">
        <out xmlns:foo="http://www.example.com/foo">{$foo:var}</out>
    </xsl:template>
    
</xsl:stylesheet>