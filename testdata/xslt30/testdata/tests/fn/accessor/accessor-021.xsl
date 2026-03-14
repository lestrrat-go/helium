<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of fn:data accessor with no schema validation, context node set to document, element, attr, txt, comment, and PI nodes -->

   <xslt:template match="/">
      <out>
         <doc_node>doc_typed_value="<xslt:value-of select="if (data(.) instance of xs:untypedAtomic) then data(.) else 'WROONG TYPE returned'"/>"</doc_node>
         <xslt:apply-templates select="*"/>
         <xslt:text>|done|</xslt:text>
      </out>
   </xslt:template>

   <xslt:template match="doc">
      <xslt:text>
</xslt:text>
      <Elmt>typed_value="<xslt:value-of select="if (data(.) instance of xs:untypedAtomic) then data(.) else 'WROONG TYPE returned'"/>"</Elmt>
      <xslt:apply-templates select="@*|node()"/>
      <xslt:text>|up|</xslt:text>
      <xslt:for-each select="namespace::*[name(.)='sub']">
	        <xslt:value-of select="if (data(.) instance of xs:string) then data(.) else 'WROONG TYPE returned'"/>
      </xslt:for-each>
   </xslt:template>

   <xslt:template match="element()">
      <xslt:text>
</xslt:text>
      <Elmt>typed_value="<xslt:value-of select="if (data(.) instance of xs:untypedAtomic) then data(.) else 'WROONG TYPE returned'"/>"</Elmt>
      <xslt:apply-templates select="@*|node()"/>
      <xslt:text>|up|</xslt:text>
   </xslt:template>

   <xslt:template match="attribute()">
      <Attr>typed_value="<xslt:value-of select="if (data(.) instance of xs:untypedAtomic) then data(.) else 'WROONG TYPE returned'"/>"</Attr>
   </xslt:template>

   <xslt:template match="processing-instruction()">
      <PI>typed_value="<xslt:value-of select="if (data(.) instance of xs:string) then data(.) else 'WROONG TYPE returned'"/>"</PI>
   </xslt:template>

   <xslt:template match="comment()">
      <Cmt>typed_value="<xslt:value-of select="if (data(.) instance of xs:string) then data(.) else 'WROONG TYPE returned'"/>"</Cmt>
   </xslt:template>

   <xslt:template match="text()">
  
      <Txt>typed_value="<xslt:value-of select="if (data(.) instance of xs:untypedAtomic) then data(.) else 'WROONG TYPE returned'"/>"</Txt>
   </xslt:template>
</xslt:transform>
