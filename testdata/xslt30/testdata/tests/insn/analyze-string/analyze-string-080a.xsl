<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
				version="2.0">

  <!-- FileName: regex20_035a -->
  <!-- Document: http://www.w3.org/TR/xslt20/ -->
  <!-- DocVersion: 20051103 -->
  <!-- Section: 15.2 regex-group() -->
  <!-- Creator: Vesela Parapounska -->
  <!-- Purpose: Stylesheet to be imported into regex20_035.xsl-->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="str1">
 	<xsl:value-of select="regex-group(1)"/>
</xsl:template>

  <!-- Copyright IBM Corp. 2004, 2005. -->

</xsl:stylesheet>
