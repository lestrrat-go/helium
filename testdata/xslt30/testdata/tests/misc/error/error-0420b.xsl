<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                                 error if the result sequence used to construct the
                              content of a document node contains a namespace node or attribute
                              node.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:my="http://my.uri/"
                version="2.0">

<?error XTDE0420?>



  <xsl:template name="main">
      <out>
         <xsl:variable name="doc">
            <xsl:attribute name="ginger"/>
         </xsl:variable>
         <xsl:copy-of select="$doc"/>  
      </out>
  </xsl:template>
  


</xsl:stylesheet>
