<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                           error if the collation attribute of
                           xsl:sort (after resolving against the base URI) is not a
                        URI that is recognized by the implementation as referring to a
                        collation.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE1035?>


  <xsl:template name="main">
      <out>
         <xsl:for-each select="1 to 5">
            <xsl:sort select="."
                      collation="http://a.collation.that.noone.is.likely.to.recognize/"/>
            <xsl:value-of select="."/>
         </xsl:for-each>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
