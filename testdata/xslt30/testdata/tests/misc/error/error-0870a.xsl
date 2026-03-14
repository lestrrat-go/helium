<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the
                           select attribute of the xsl:value-of
                        element is present when the content of the element is non-empty, or if the
                           select attribute is absent when the content is empty.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0870?>

   <xsl:param name="p" select="'ns'"/>

  <xsl:template name="main">
      <my:out>
        <xsl:value-of select="3">four</xsl:value-of>
      </my:out>
  </xsl:template>
  



</xsl:stylesheet>
