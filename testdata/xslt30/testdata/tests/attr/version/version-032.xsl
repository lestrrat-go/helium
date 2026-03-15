<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" 
    xmlns:ext="http://somebody.elses.extension" 
    extension-element-prefixes="ext">
    
    <!-- Test xsl:fallback with an XSLT instruction as a child. -->
    
    <xsl:template match="/">
        <out>
            <ext:test>
                <xsl:fallback>
                    <xsl:text>Fallback: extension was not found.</xsl:text>
                </xsl:fallback>
            </ext:test>
        </out>
    </xsl:template>
    
</xsl:stylesheet>