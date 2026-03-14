<?xml version="1.0" encoding="UTF-8"?>
<!--In general, a circularity in a stylesheet is a non-recoverable dynamic error.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE0640?>

  <xsl:param name="ping" select="$pong+2"/>   

  <xsl:template name="main">
      <out>
        <xsl:value-of select="$pong"/>
      </out>
  </xsl:template>
  
  <xsl:variable name="pong" select="$ping - 2"/>   

  


</xsl:stylesheet>
