<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test type of global xsl:variable,  @as="item()+" and @select contains one of:
				- sequence of xs:untypedAtomic from input file, explicitly created built-in 
				primitive type (xs:double), mixed sequence of explicitly created built-in primitive
				types, attribute() from input file, document-node() from input file, temporary tree,
				literal string -->

   <xslt:variable name="temporary-tree">
      <a>
         <b att="one">hello</b>
      </a>
   </xslt:variable>

   <xslt:variable name="var1" select="/doc//text()" as="item()+"/>

   <xslt:variable name="var2" select="xs:double(2.0)" as="item()+"/>

   <xslt:variable name="var3"
      select="(xs:string('my'), xs:integer(5), xs:anyURI('http://hello.org'))" as="item()+"/>

   <xslt:variable name="var4" select="/doc/item/attribute()" as="item()+"/>

   <xslt:variable name="var5" select="/" as="item()+"/>

   <xslt:variable name="var6" select="$temporary-tree" as="item()+"/>

   <xslt:variable name="var7" select="'mystring'" as="item()+"/>

   <xslt:template match="/doc">
      <out>
         <e1>
            <xslt:value-of select="$var1 instance of item()+"/>
         </e1>
         <e2>
            <xslt:value-of select="$var2 instance of item()+"/>
         </e2>
         <e3>
            <xslt:value-of select="$var3 instance of item()+"/>
         </e3>
         <e4>
            <xslt:value-of select="$var4 instance of item()+"/>
         </e4>
         <e5>
            <xslt:value-of select="$var5 instance of item()+"/>
         </e5>
         <e6>
            <xslt:value-of select="$var6 instance of item()+"/>
         </e6>
         <e7>
            <xslt:value-of select="$var7 instance of item()+"/>
         </e7>
      </out>
   </xslt:template>
</xslt:transform>
