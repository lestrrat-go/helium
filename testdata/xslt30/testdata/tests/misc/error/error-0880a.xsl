<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the
                        select attribute of the
                        xsl:processing-instruction element is present unless the
                     element has empty content.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0880?>

   <xsl:param name="p" select="'ns'"/>

  <xsl:template name="main">
      <my:out>
        <xsl:processing-instruction name="pi" select="3">four</xsl:processing-instruction>
      </my:out>
  </xsl:template>
  



</xsl:stylesheet>
