<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes=" xs" version="3.0">

  <!-- streaming, xsl:iterate, //a/b selection: 
       not guaranteed streamable in spec, streamable in Saxon -->

  <xsl:strip-space elements="*"/>


  <xsl:template name="main">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:iterate select="/*/*/ITEM/TITLE">
           <title><xsl:value-of select="."/></title>
        </xsl:iterate>  
      </xsl:source-document>
    </out>
  </xsl:template>

</xsl:transform>
