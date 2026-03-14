<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#attribute-value-templates?>
  <!-- PURPOSE: Check space-separation of a list-valued AVT (new 2.0 incompatibility). -->
  <!-- AUTHOR: Michael Kay -->
  <!--test changed in 7.4-->
  <xsl:template match="doc">
  <out>
    <a val="[{(1,2,3)}]" simple="{(1,2,3)}"/>
    <b val="[{item}]" simple="{item}"/>
  </out>
  </xsl:template>
 
</xsl:stylesheet>
