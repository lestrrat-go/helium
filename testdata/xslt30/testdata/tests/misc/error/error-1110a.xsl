<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error if the collation URI specified to
                        xsl:for-each-group (after resolving against the base URI)
                     is a collation that is not recognized by the implementation. (For notes,
                        .)-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE1110?>


  <xsl:template name="main">
      <xsl:variable name="pop">
         <a/>
         <b/>
         <c/>
      </xsl:variable>
      <out>
         <xsl:for-each-group select="$pop/*"
                             group-adjacent="local-name()"
                             collation="http://unknown.collation.uri/">
            <xsl:value-of select="current-group()"/>
         </xsl:for-each-group>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
