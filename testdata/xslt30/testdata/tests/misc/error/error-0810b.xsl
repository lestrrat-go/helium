<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if 
                        within a package there is
                        more than one such declaration more than one
                              xsl:namespace-alias declaration with
                        the same literal namespace
                           URI and the same import
                           precedence and different values for the target namespace URI, unless
                        there is also an xsl:namespace-alias declaration with the
                        same literal namespace URI
                        and a higher import precedence. -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                xmlns:your="http://my.com/"
                version="2.0">

<?error XTSE0810?>



  <xsl:template name="main">
      <my:out>
        <xsl:sequence select="2"/>
      </my:out>
  </xsl:template>
  
   <xsl:namespace-alias stylesheet-prefix="my" result-prefix="xs"/>
   <xsl:namespace-alias stylesheet-prefix="your" result-prefix="xml"/>


</xsl:stylesheet>
