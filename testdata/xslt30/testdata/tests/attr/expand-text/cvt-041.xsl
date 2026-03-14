<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- now permitted under resolution of bug 20874 -->

    <xsl:template name="main" expand-text="yes">
        <out>{array{1,2,3}}</out>
    </xsl:template>
    
</xsl:stylesheet>