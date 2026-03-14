<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- A sequence of merge-input is 'text', but the @data-type
        attribute of xsl:merge-key is 'number' -->
    
    <xsl:output method="xml" indent="no"/>
    <xsl:template match="/">
       
        <results>
            <xsl:merge>
                <xsl:merge-source select="doc('america.xml')/america/country">
                        <xsl:merge-key order="ascending" data-type='text'>
                            <xsl:sequence select="."/>
                        </xsl:merge-key>
                </xsl:merge-source>
                
                <xsl:merge-action>
                    <xsl:copy-of select="current-merge-group()"/>                    
                </xsl:merge-action>
            </xsl:merge>
        </results>
        
    </xsl:template>
</xsl:stylesheet>