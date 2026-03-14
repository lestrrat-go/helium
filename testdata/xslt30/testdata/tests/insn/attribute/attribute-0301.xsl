<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

 <xsl:output method="html"/>

 <?spec xslt#creating-attributes?>
 <!-- test use of non-ASCII characters in URL and non-URL attributes -->
 <!-- using HTML output method -->

 <xsl:template match="page">
  <html>
   <title>test</title>
   <body>
    <xsl:apply-templates/>
   </body>
  </html>
 </xsl:template>

 <xsl:template match="test">
  <a><xsl:attribute name="href">http://<xsl:apply-templates/></xsl:attribute>url</a>
 </xsl:template>

 <xsl:template match="formfield">
  <input type="text" name="{@name}" value="{.}"/>
 </xsl:template>

 <xsl:template match="formfield2">
  <input type="text" name="{@name}">
   <xsl:attribute name="value">
    <xsl:apply-templates/>
   </xsl:attribute>
  </input>
 </xsl:template>

 <xsl:template match="formtxt">
  <textarea name="{@name}" wrap="virtual" rows="4" cols="40">
   <xsl:apply-templates/>
  </textarea>
 </xsl:template>

</xsl:stylesheet>
