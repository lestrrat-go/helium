<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- An error should be raised as the @select attribute of xsl:merge-key should be a singleton,
    which it is evaluated to be a sequence of items -->
    
    <xsl:output method="xml" indent="no"/>
    <xsl:template match="/">
        
        <results>
            <xsl:merge>
                <xsl:merge-source select="doc('america.xml')/america">
                    
                        <xsl:merge-key select="country" order="ascending"/>
                    
                </xsl:merge-source>
                <xsl:merge-action>
                    <xsl:value-of select="current-merge-group()"/>                    
                </xsl:merge-action>
            </xsl:merge>
        </results>
        
    </xsl:template>
</xsl:stylesheet>