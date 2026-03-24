<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:axsl="http://axsl.uri/"
 version="2.0">

 <!-- BUG: Saxon 5.2/020 -->
 <!-- DESCRIPTION: xsl:namespace-alias must come before the relevant xsl:template elements -->




 <xsl:template match="/">

  <axsl:stylesheet version="1.0">
   <axsl:variable name="temp" select="3"/>
  </axsl:stylesheet>

 </xsl:template>

 <xsl:namespace-alias stylesheet-prefix="axsl" result-prefix="xsl"/>

</xsl:transform>
