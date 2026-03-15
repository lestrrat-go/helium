<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs"
                default-validation="preserve">

   <!-- Test that pattern "attribute(A,T)" gets a default priority of 0.25. -->

   <xslt:output method="xml" indent="no" encoding="UTF-8"/>

   <xslt:variable name="tree">
      <e1>
         <xslt:attribute name="x" type="xs:integer">1</xslt:attribute>
      </e1>
      <e2>
         <xslt:attribute name="x" type="xs:string">E2-wrong</xslt:attribute>
      </e2>
      <e3>
         <xslt:attribute name="y" type="xs:integer">39025</xslt:attribute>
      </e3>
      <e4>
         <xslt:attribute name="x" type="xs:integer">4444</xslt:attribute>
      </e4>
      <e5>
         <xslt:attribute name="x" type="xs:string">E5-wrong</xslt:attribute>
         <xslt:attribute name="y" type="xs:integer">5000005</xslt:attribute>
      </e5>
      <e6>
         <xslt:attribute name="z" type="xs:string">E6-wrong</xslt:attribute>
      </e6>
   </xslt:variable>

   <xslt:template match="doc">
      <out>
         <xslt:apply-templates select="$tree/*/@x"/>
      </out>
   </xslt:template>

   <xslt:template match="attribute(x,xs:integer)">
      <xslt:text>!Match-of-attribute-x-integer:</xslt:text>
      <xslt:value-of select="name(..)"/>
   </xslt:template>

   <xslt:template match="@*" priority="0.2">
      <xslt:text>!Match-any-attribute:</xslt:text>
      <xslt:value-of select="name(..)"/>
   </xslt:template>

   <xslt:template match="e4/@x" priority="0.3">
      <xslt:text>!Match-e4-x</xslt:text>
   </xslt:template>
</xslt:transform>
