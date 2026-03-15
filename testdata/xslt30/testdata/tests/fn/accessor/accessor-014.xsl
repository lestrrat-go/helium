<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of nilled() accessor with document, element, attr, txt, comment, and PI nodes. -->

   <xslt:template match="/">
      <out>
         <doc_node>doc_name="<xslt:value-of select="nilled(.)"/>"
    </doc_node>
         <xslt:apply-templates select="*"/>
         <xslt:text>|done|</xslt:text>
      </out>
   </xslt:template>

   <xslt:template match="doc">
      <xslt:text>
</xslt:text>
      <Elmt>name="<xslt:value-of select="nilled(.)"/>"
  </Elmt>
      <xslt:apply-templates select="@*|node()"/>
      <xslt:text>|up|</xslt:text>
      <xslt:for-each select="namespace::*">
         <xslt:value-of select="nilled(.)"/>
      </xslt:for-each>
   </xslt:template>

   <xslt:template match="element()">
      <xslt:text>
</xslt:text>
      <Elmt>name="<xslt:value-of select="nilled(.)"/>"
  </Elmt>
      <xslt:apply-templates select="@*|node()"/>
      <xslt:text>|up|</xslt:text>
   </xslt:template>

   <xslt:template match="processing-instruction()">
      <PI>name="<xslt:value-of select="nilled(.)"/>" 
  </PI>
   </xslt:template>

   <xslt:template match="attribute()">
      <Attr>name="<xslt:value-of select="nilled(.)"/>"
  </Attr>
   </xslt:template>

   <xslt:template match="text()">
  
      <Txt>name="<xslt:value-of select="nilled(.)"/>"
  </Txt>
   </xslt:template>

   <xslt:template match="comment()">
      <Cmt>name="<xslt:value-of select="nilled(.)"/>"
  </Cmt>
   </xslt:template>
</xslt:transform>
