<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error to specify the collation attribute 
                     or the composite attribute if neither the
                        group-by attribute nor group-adjacent attribute is
                     specified.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1090?>


  <xsl:template name="main">
      <xsl:variable name="pop">
         <a/>
         <b/>
         <c/>
      </xsl:variable>
      <out>
         <xsl:for-each-group select="$pop/*"
                             group-starting-with="a"
                             collation="http://www.w3.org/2005/xpath-functions/collation/codepoint">
            <xsl:value-of select="current-group()"/>
         </xsl:for-each-group>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
