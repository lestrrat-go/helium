<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://www.mytest.net"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Test type of result of xsl:function that has @as="element(QName, xs:untyped)" 
				and sequence constructor contains xsl:element or an LRE.-->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:function name="my:func1" as="element(my:elem, xs:untyped)">
	     <my:elem>
		my element
	</my:elem>
   </xslt:function>

   <xslt:function name="my:func2" as="element(item, xs:untyped)">
	     <xslt:element name="item">hello</xslt:element>
   </xslt:function>

   <xslt:template match="/doc">
      <out>
         <xslt:value-of select="my:func1() instance of element(my:elem, xs:untyped)"/>
         <xslt:value-of select="my:func2() instance of element(item, xs:untyped)"/>
      </out>
   </xslt:template>
</xslt:transform>
