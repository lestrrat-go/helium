<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                                 error if the result sequence used to construct the
                              content of an element node contains a namespace node or attribute node
                              that is preceded in the sequence by a node that is neither a namespace
                              node nor an attribute node.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:my="http://my.uri/"
                version="2.0">

<?error XTDE0410?>



  <xsl:template name="main">
      <out>
      fred
      <xsl:attribute name="ginger"/>
      </out>
  </xsl:template>
  


</xsl:stylesheet>
