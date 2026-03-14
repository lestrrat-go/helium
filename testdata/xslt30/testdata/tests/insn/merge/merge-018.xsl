<?xml version="1.0"?> 
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
    xmlns:s="http://s.com/" exclude-result-prefixes="s">
    
    <!-- PURPOSE: Test xsl:merge - perform sort with a xsl:merge-input  -->
    
    <xsl:output indent="no"/>
    
    <xsl:template match="/">
        <out>
            <xsl:merge>
                
                <xsl:merge-source 
                	select="doc('europe.xml')/europe/country" 
                	sort-before-merge="yes">
                        <xsl:merge-key select="." />
                </xsl:merge-source>
                
                
                <xsl:merge-source select="doc('america.xml')/america/country">
                    <xsl:merge-key select="." />          
                </xsl:merge-source>
                <xsl:merge-action>
                        <xsl:copy-of select="current-merge-group()"/>
                </xsl:merge-action>
            </xsl:merge>
        </out>
        
       
    </xsl:template>
</xsl:stylesheet>
