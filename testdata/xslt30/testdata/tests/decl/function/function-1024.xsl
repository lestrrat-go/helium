<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
 <xsl:output method="xml" encoding="UTF-8" indent="yes" omit-xml-declaration="yes" />
 
  
 <xsl:template match="/">
  <!-- Testing fn:format-integer#2 with error case-->
  <?spec fo#func-format-integer?>
  <out>
  <a><xsl:value-of select="format-integer(123 ,'0#')"/></a>
  </out>
  </xsl:template>
</xsl:stylesheet>
