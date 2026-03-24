<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xslt="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Tests use of a document node inside an instruction that is constructing simple content. -->

   <xslt:template match="/">
      <xslt:variable name="v1">
         <xslt:value-of select="''"/>
      </xslt:variable>
      <xslt:variable name="v2">
         <xslt:text/>
      </xslt:variable>
      <out>
         <xslt:comment>
            <xslt:document>
               <xslt:sequence xmlns:xs="http://www.w3.org/2001/XMLSchema"
                  select="doc/ch, doc//comment(), xs:integer('9456798'), xs:dayTimeDuration('P020DT03H')"/>
               <xslt:value-of separator="*"
                  select="('Old', '', 'est', $v1, 'att', '', 'rib', $v2,'ute')"/>
            </xslt:document>
         </xslt:comment>
      </out>
   </xslt:template>
   
</xslt:transform>
