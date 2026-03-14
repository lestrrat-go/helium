<?xml version="1.0"?> 
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
    xmlns:s="http://s.com/" exclude-result-prefixes="s">
    
    <!-- PURPOSE: Test xsl:merge-key context: visibility of variable defined in xsl:merge-input  -->
    <!-- NOTE 2018-05-12: changes to the spec mean the original purpose of this test has been lost -->
    
    <xsl:output indent="no"/>
    
    <xsl:template match="/">
        <out>
            <xsl:merge>
                
                <xsl:merge-source>
                    <xsl:merge-input>
                        <xsl:variable name="o">ascending</xsl:variable>
                        
                        <xsl:perform-sort select="doc('europe.xml')/europe/country">
                           
                            <xsl:sort select="."/>
                        </xsl:perform-sort>
                        
                        <xsl:merge-key select="." order="{$o}"/>
                    </xsl:merge-input>
                </xsl:merge-source>
                
                <xsl:merge-action>
                        <xsl:copy-of select="current-merge-group()"/>
                </xsl:merge-action>
            </xsl:merge>
        </out>
        
       
    </xsl:template>
</xsl:stylesheet>
