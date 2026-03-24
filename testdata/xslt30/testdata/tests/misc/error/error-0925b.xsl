<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error if the xsl:namespace instruction generates
                     a namespace node whose name is xml and whose string value is not
                        http://www.w3.org/XML/1998/namespace, or a namespace node whose
                     string value is http://www.w3.org/XML/1998/namespace and whose
                     name is not xml.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE0925?>

   <xsl:param name="p" select="'yml'"/>

  <xsl:template name="main">
      <my:out>
        <xsl:namespace name="{$p}">http://www.w3.org/XML/1998/namespace</xsl:namespace>
      </my:out>
  </xsl:template>
  



</xsl:stylesheet>
