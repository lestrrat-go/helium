<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test document() function. Argument is a sequence of xs:untypedAtomic:
				-a sequence of nodes
				-a sequence of variables
				-mixed sequence of strings, nodes and variables
				-a subsequence
				Passed values represent relative URI references and are resolved against
				the base-uri property from the static context.  -->

   <t:variable name="ea" select="/doc/str1"/>

   <t:variable name="doc" select="//str3/@href"/>

   <t:strip-space elements="*"/>

   <t:template name="a">
      <a>
         <t:copy-of select="doc/document((str1,str2,str3,str4))//(body[1]|out)"/>
      </a>
   </t:template>

   <t:template name="b">
      <b>
         <t:copy-of select="document(($ea, $doc))//(body[1]|out)"/>
      </b>
   </t:template>

   <t:template name="c">
      <c>
         <t:copy-of
            select="doc/document((xs:untypedAtomic('doc02.xml'), str2, str3/@href,'doc04.xml',$ea))//(body[1]|out)"/>
      </c>
   </t:template>

   <t:template name="d">
      <d>
         <t:copy-of
            select="doc/document(subsequence((str1, str2, str3/@href,'doc04.xml',$ea), 2,1))//(body[1]|out)"/>
      </d>
   </t:template>

</t:transform>
