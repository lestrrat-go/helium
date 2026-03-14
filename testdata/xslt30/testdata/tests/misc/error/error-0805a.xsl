<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an
                        attribute on a literal result element is in the XSLT namespace, unless it is one of the
                        attributes explicitly defined in this specification.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0805?>



  <xsl:template name="main">
      <out xsl:banana="3">
        <xsl:sequence select="2"/>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
