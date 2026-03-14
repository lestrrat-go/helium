<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test type of global xsl:variable selecting an attribute 
				node from input file or from a temporary tree, @as="attribute(QName, xs:untypedAtomic)". -->

   <xslt:variable name="temporary-tree">
      <a>
         <b att="one">hello</b>
      </a>
   </xslt:variable>

   <xslt:variable name="var1" select="/doc/item/@attrib" as="attribute(attrib, xs:untypedAtomic)"/>

   <xslt:variable name="var2" select="$temporary-tree/a/b/@att"
      as="attribute(att, xs:untypedAtomic)"/>

   <xslt:template match="/doc">
      <out>
         <e1>
            <xslt:value-of select="$var1 instance of attribute(attrib, xs:untypedAtomic)"/>
         </e1>
         <e2>
            <xslt:value-of select="$var2 instance of attribute(att, xs:untypedAtomic)"/>
         </e2>
      </out>
   </xslt:template>
</xslt:transform>
