<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:example1="http://www.example1.com"
                xmlns:example2="http://www.example2.com"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs example1 example2">
<!-- Purpose: Tests xs:QName types in a sequence -->

   <xslt:variable name="qnames" as="item()*">
      <xslt:sequence select="(xs:untypedAtomic('example1:foo'),doc/qname,xs:QName('example2:foo'))"/>
   </xslt:variable>

   <xslt:template match="/">
      <out>
         <xslt:text>
</xslt:text>
         <qname>
            <xslt:value-of select="$qnames[0] instance of xs:QName"/>
         </qname>
         <xslt:text>
</xslt:text>
         <qname>
            <xslt:value-of select="$qnames[1] instance of xs:QName"/>
         </qname>
         <xslt:text>
</xslt:text>
         <qname>
            <xslt:value-of select="$qnames[2] instance of xs:QName"/>
         </qname>
         <xslt:text>
</xslt:text>
         <qname>
            <xslt:value-of select="$qnames[3] instance of xs:QName"/>
         </qname>
         <xslt:text>
</xslt:text>
         <qname>
            <xslt:value-of select="$qnames[4] instance of xs:QName"/>
         </qname>
         <xslt:text>
</xslt:text>
      </out>
   </xslt:template>
</xslt:transform>
