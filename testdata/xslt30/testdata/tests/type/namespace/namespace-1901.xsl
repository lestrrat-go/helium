<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:b="http://test.com/b"
    xmlns:c="http://test.com/c" 
    exclude-result-prefixes="#all" version="3.0">
    
    <xsl:output method="xml" indent="no"/>

    <!-- Test that the no-namespace "header" element is serialized correctly
         with an xmlns="" undeclaration. Saxon bug 3889 -->
 
    <xsl:template match="/*">
        <xsl:copy>
            <xsl:sequence select="doc('namespace-19b.xml')/b:request"/>
        </xsl:copy>
    </xsl:template>
    
</xsl:stylesheet>
