<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the
                              value #default is used within the
                                 [xsl:]exclude-result-prefixes attribute and the parent
                              element of the [xsl:]exclude-result-prefixes attribute
                              has no default namespace.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0809?>



  <xsl:template name="main">
      <out xsl:exclude-result-prefixes="my #default">
        <xsl:sequence select="2"/>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
