<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if a
                              namespace prefix is used within the
                                 [xsl:]exclude-result-prefixes attribute and there is
                              no namespace binding in scope for that prefix.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0808?>



  <xsl:template name="main">
      <out xsl:exclude-result-prefixes="my your">
        <xsl:sequence select="2"/>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
