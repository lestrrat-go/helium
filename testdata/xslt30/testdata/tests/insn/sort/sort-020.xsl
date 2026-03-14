<?xml version="1.0"?> 
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- PURPOSE: Test xsl:sort using a code-point collation -->



<xsl:template match="/">
   <out>
   <xsl:variable name="s" as="element()*">
     <xsl:perform-sort select="//num">
       <xsl:sort select="." collation="http://www.w3.org/2005/xpath-functions/collation/codepoint"/>
     </xsl:perform-sort>
    </xsl:variable>
    <xsl:value-of select="$s" separator=","/>
   </out>
</xsl:template>
</xsl:stylesheet>
