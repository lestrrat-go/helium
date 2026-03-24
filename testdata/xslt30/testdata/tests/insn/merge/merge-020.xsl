<?xml version="1.0"?> 
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
    xmlns:s="http://s.com/" exclude-result-prefixes="s">
    
    <!-- PURPOSE: incompatible merge keys  -->
    
    
    <xsl:function name="s:sk1">
        <xsl:param name="sort-key" as="function(*)"/>
        <xsl:merge>
            <xsl:merge-source  select="doc('europe.xml')/europe/country">

                    <xsl:merge-key select="." data-type="text" collation="http://www.w3.org/2005/xpath-functions/collation/codepoint" />
            </xsl:merge-source>
            
            
            <xsl:merge-source select="doc('america.xml')/america/country">
                    
                    <xsl:merge-key select="." data-type="number" collation="http://www.w3.org/2005/xpath-functions/collation/codepoint" />          
                
            </xsl:merge-source>
            <xsl:merge-action>
                <out>
                    <xsl:value-of select="."/>
                </out>
            </xsl:merge-action>
        </xsl:merge>
    </xsl:function>
    
    <xsl:output indent="yes" encoding="iso-8859-1"/>
    
    <xsl:template match="/">
        <out lang="default-order">
            <xsl:copy-of select="s:sk1(function($n as node()) {$n})"/> <!--separator=","/> -->
        </out>

    </xsl:template>
</xsl:stylesheet>
