<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test for generating two or more result trees with same URI where the first tree has an explicit 
				xsl:result-document with href="" and the second tree is implicit -->

   <t:output method="text"/>

   <t:template match="/">
      <t:result-document href="">
To the primary result tree.
</t:result-document>
The primary result tree is closed, 
this text is not under any result document 
so this should cause problems.
</t:template>
</t:transform>
