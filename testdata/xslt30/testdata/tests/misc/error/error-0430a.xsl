<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                                 error if the result sequence contains two or more
                              namespace nodes having the same name but different string values (that is, namespace
                              nodes that map the same prefix to different namespace URIs).-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:my="http://my.uri/"
                version="2.0">

<?error XTDE0430?>



  <xsl:template name="main">
      <out>
         <xsl:namespace name="n">tweedledee</xsl:namespace>
         <xsl:namespace name="n">tweedledum</xsl:namespace>
      </out>
  </xsl:template>
  


</xsl:stylesheet>
