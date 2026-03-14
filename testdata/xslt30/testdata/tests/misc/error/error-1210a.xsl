<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the xsl:key declaration has a
                           collation attribute whose value (after resolving against the
                        base URI) is not a URI recognized by the implementation as referring to a
                        collation.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1210?>

   <xsl:key name="k"
            match="*"
            use="17"
            collation="http://unknown.collation.uri/"/>


  <xsl:template name="main">
      <out>
         <xsl:sequence select="3"/>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
