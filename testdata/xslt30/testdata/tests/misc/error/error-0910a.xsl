<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the
                        select attribute of the xsl:namespace element
                     is present when the element has content other than one or more
                        xsl:fallback instructions, or if the select
                     attribute is absent when the element has empty content.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0910?>

   <xsl:param name="p" select="'ns'"/>

  <xsl:template name="main">
      <my:out>
        <xsl:namespace name="ns" select="'http://www.your.uri/'">http://my.com/</xsl:namespace>
      </my:out>
  </xsl:template>
  



</xsl:stylesheet>
