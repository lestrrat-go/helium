<?xml version="1.0"?> 
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
    xmlns:s="http://s.com/" exclude-result-prefixes="s">
    
    <xsl:output indent="yes" encoding="iso-8859-1"/>
    
    <xsl:template match="/">
        <xsl:merge>
            <xsl:merge-source select="doc('europe.xml')/europe/country">
                    
                    <xsl:merge-key select="." data-type="text" collation="http://www.w3.org/2005/xpath-functions/collation/codepoint" />

            </xsl:merge-source>
            
            
            <xsl:merge-source select="doc('america.xml')/america/country">
                    
                    <xsl:merge-key select="." data-type="text" order="ascending" collation="http://www.w3.org/2005/xpath-functions/collation/codepoint" />          

                
            </xsl:merge-source>
            <xsl:merge-action>
                <out>
                    <xsl:value-of select="."/>
                </out>
            </xsl:merge-action>
        </xsl:merge>

        
       
    </xsl:template>
</xsl:stylesheet>
