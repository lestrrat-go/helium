<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://www.mytest.net"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with global xsl:variable that has @as="attribute(QName, xs:untypedAtomic)+" and its sequence constructor contains:
				-one attribute node, a sequence of attribute nodes from input file, attribute nodes from a temporary tree -->

   <xslt:variable name="temporary-tree">
      <a att="one">
         <a att="two">hello</a>
         <b att="three">hello</b>
      </a>
   </xslt:variable>

   <xslt:variable name="var1" as="attribute(my:elem, xs:untypedAtomic)+">
      <xslt:attribute name="my:elem">hello</xslt:attribute>
   </xslt:variable>

   <xslt:variable name="var2" as="attribute(attrib, xs:untypedAtomic)+">
      <xslt:copy-of select="/doc//@*"/>
   </xslt:variable>

   <xslt:variable name="var3" as="attribute(att, xs:untypedAtomic)+">
      <xslt:copy-of select="$temporary-tree//@att"/>
   </xslt:variable>

   <xslt:template match="/doc">
      <out>
         <e1>
            <xslt:value-of select="$var1 instance of attribute(my:elem, xs:untypedAtomic)+"/>
            <xslt:value-of select="$var1 instance of attribute(my:elem, xs:untypedAtomic)"/>
         </e1>
         <e2>
            <xslt:value-of select="$var2 instance of attribute(attrib, xs:untypedAtomic)+"/>
            <xslt:value-of select="$var2 instance of attribute(attrib, xs:untypedAtomic)"/>
         </e2>
         <e3>
            <xslt:value-of select="$var3 instance of attribute(att, xs:untypedAtomic)+"/>
            <xslt:value-of select="$var3 instance of attribute(att, xs:untypedAtomic)"/>
         </e3>
      </out>
   </xslt:template>
</xslt:transform>
