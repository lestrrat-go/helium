<?xml version="1.0" ?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0">

<?spec xslt#xsl-if?>
<!-- BUG 6.0/006 -->
<!-- DESCRIPTION: Zero-length text nodes are created in an empty element -->

<xsl:template match="icon">
<out note="output changed in 7.9">
   <xsl:if test="count(.//text())>0"> 
    <a>bug</a>
   </xsl:if>
</out>
</xsl:template>


</xsl:stylesheet >

