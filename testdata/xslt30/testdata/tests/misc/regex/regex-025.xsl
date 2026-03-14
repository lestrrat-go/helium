<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test reference to a group that doesn't exist in replace() -->
  <?spec fo#func-replace?>
  
  <xsl:output encoding="iso-8859-1"/>
  
  <xsl:template match="/">
     <xsl:variable name="t">He said, "I don't" eat 'grass'</xsl:variable>
     <xsl:variable name="p">(['"])(.*?)\1</xsl:variable>
     <a><xsl:value-of select="replace($t, $p, '[$2]#$5#')"/></a>
  </xsl:template>

</xsl:stylesheet>