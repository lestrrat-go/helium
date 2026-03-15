<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:t="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test document() function. Argument is a sequence of xs:string:
				-a sequence of literal strings, 
				-a sequence of variables, 
				-mixed sequence of strings, nodes and variables, 
				-a subsequence
				Passed values represent relative URI references and are resolved 
				against the base-uri property from the static context. -->
   
   <!-- Note that the results of the document() function with multiple strings as arguments is in document
        order, which is unpredictable -->

   <t:variable name="ea" select="xs:string('doc02.xml')"/>

   <t:variable name="doc" select="xs:string('doc01.xsl')"/>
   
   <t:strip-space elements="*"/>
   
   <t:template name="a">
      <a>
         <t:copy-of select="document(('doc02.xml','doc03.xml','doc01.xsl','doc04.xml'))//(body[1]|out)"/>
      </a>
   </t:template>
   
   <t:template name="b">
      <b>
         <t:copy-of select="document(($ea, $doc))//(body[1]|out)"/>
      </b>
   </t:template>
   
   <t:template name="c">
      <c>
         <t:copy-of select="document((xs:string('doc02.xml'),xs:string(//str2/@href),xs:string(//str3/@href),'doc04.xml',$ea))//(body[1]|out)"/>
      </c>
   </t:template>
   
   <t:template name="d">
      <d>
         <t:copy-of select="document(subsequence((xs:string('doc02.xml'),xs:string(//str2/@href),//str3/@href,'doc04.xml',$ea), 2,1))//(body[1]|out)"/>
      </d>
   </t:template>
   
</t:transform>
