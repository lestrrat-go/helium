<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://www.examples.com/my"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose:Use xsl:perform-sort inside xsl:function to sort a sequence of atomic values
   (Idea taken from Specification's examples). -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:template match="/">
      <xslt:param name="list1" select="('a','f','u','d','z')" as="xs:anyAtomicType*"/>
      <xslt:param name="list2" select="(003,-5, 0.0, 0.002, 10, 400, 3)" as="xs:anyAtomicType*"/>
      <out>
         <letter>
            <xslt:value-of select="my:func($list1)" separator=" | "/>
         </letter>
         <number>
            <xslt:value-of select="my:func($list2)" separator=" | "/>
         </number>
      </out>
   </xslt:template>

   <xslt:function name="my:func" as="xs:anyAtomicType*">
      <xslt:param name="par" as="xs:anyAtomicType*"/>
      <xslt:perform-sort select="$par">
         <xslt:sort select="."/>
      </xslt:perform-sort>
   </xslt:function>

   <xslt:template match="text()"/>
</xslt:transform>
