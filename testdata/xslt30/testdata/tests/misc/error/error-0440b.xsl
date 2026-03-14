<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                                 error if the result sequence contains a namespace node
                              with no name and the element node being constructed has a null
                              namespace URI (that is, it is an error to define a default namespace
                              when the element is in no namespace). -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:my="http://my.uri/"
                version="2.0">

<?error XTDE0440?>



  <xsl:template name="main">
      <out>
         <xsl:namespace name="">tweedledee</xsl:namespace>
      </out>
  </xsl:template>
  


</xsl:stylesheet>
