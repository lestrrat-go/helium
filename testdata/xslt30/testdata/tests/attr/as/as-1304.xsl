<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://www.mytest.net"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with global xsl:variable that has some form of @as="document-node(element(*, xs:untyped?))+" 
				and @select contains a sequence of document root nodes from input file.-->

   <xslt:variable name="var1" select="root(), /, /doc/(/)" as="document-node(element(*, xs:untyped?))+"/>
   <xslt:variable name="var2" select="root(), /, /doc" as="node()+"/>

   <xslt:template match="/doc">
      <out>
         <e1>
            <xslt:value-of select="$var1 instance of document-node(element(*, xs:untyped?))?"/>
            <xslt:value-of select="$var1 instance of document-node(element(*, xs:untyped?))*"/>
            <xslt:value-of select="$var1 instance of document-node(element(*, xs:untyped?))+"/>
            <xslt:value-of select="$var1 instance of document-node(element(*, xs:untyped?))"/>
         </e1>
         <e2>
            <xslt:value-of select="$var1 instance of document-node(element(doc, xs:untyped?))?"/>
            <xslt:value-of select="$var1 instance of document-node(element(doc, xs:untyped?))*"/>
            <xslt:value-of select="$var1 instance of document-node(element(doc, xs:untyped?))+"/>
            <xslt:value-of select="$var1 instance of document-node(element(doc, xs:untyped?))"/>
         </e2>
         <e3>
            <xslt:value-of select="$var1 instance of document-node(element(nope, xs:untyped))?"/>
            <xslt:value-of select="$var1 instance of document-node(element(nope, xs:untyped))*"/>
            <xslt:value-of select="$var1 instance of document-node(element(nope, xs:untyped))+"/>
            <xslt:value-of select="$var1 instance of document-node(element(nope, xs:untyped))"/>
         </e3>
         <e4>
            <xslt:value-of select="$var1 instance of document-node(element(doc))?"/>
            <xslt:value-of select="$var1 instance of document-node(element(doc))*"/>
            <xslt:value-of select="$var1 instance of document-node(element(doc))+"/>
            <xslt:value-of select="$var1 instance of document-node(element(doc))"/>
         </e4>
         <e5>
            <xslt:value-of select="$var1 instance of document-node(element(nope))?"/>
            <xslt:value-of select="$var1 instance of document-node(element(nope))*"/>
            <xslt:value-of select="$var1 instance of document-node(element(nope))+"/>
            <xslt:value-of select="$var1 instance of document-node(element(nope))"/>
         </e5>
         <e6>
            <xslt:value-of select="$var1 instance of document-node(element(*))?"/>
            <xslt:value-of select="$var1 instance of document-node(element(*))*"/>
            <xslt:value-of select="$var1 instance of document-node(element(*))+"/>
            <xslt:value-of select="$var1 instance of document-node(element(*))"/>
         </e6>
         <e7>
            <xslt:value-of select="$var2 instance of node()?"/>
            <xslt:value-of select="$var2 instance of node()*"/>
            <xslt:value-of select="$var2 instance of node()+"/>
            <xslt:value-of select="$var2 instance of node()"/>
         </e7>
         <e8>
            <xslt:value-of select="$var2[3] instance of element(doc, xs:anyType)?"/>
            <xslt:value-of select="$var2[3] instance of element(doc, xs:anyType)*"/>
            <xslt:value-of select="$var2[3] instance of element(doc, xs:anyType)+"/>
            <xslt:value-of select="$var2[3] instance of element(doc, xs:anyType)"/>
         </e8>
      </out>
   </xslt:template>
</xslt:transform>
