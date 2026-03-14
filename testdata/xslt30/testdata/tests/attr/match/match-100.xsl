<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- Test of template matching of root node -->

<xsl:variable name="temp">
  <a>4</a>
  <b>5</b>
  <c>6</c>
</xsl:variable>

<xsl:template match="*"/>

<xsl:template match="document-node()">
  <t>
	<xsl:text>GOT_DOC_NODE</xsl:text>
	<xsl:apply-templates/>
  </t>
</xsl:template>


</xsl:stylesheet>
