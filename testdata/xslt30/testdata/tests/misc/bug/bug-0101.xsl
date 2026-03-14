<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#copying?>
 <!-- test fix for bug 5.3/001 -->

 <xsl:template match="/">
  <out>
   <xsl:variable name="tmp">
    <a a-foo="a-bar"/>
    <b b-foo="b-bar"/>
   </xsl:variable>
   <xsl:copy-of select="$tmp"/>
  </out>
 </xsl:template>



</xsl:transform>
