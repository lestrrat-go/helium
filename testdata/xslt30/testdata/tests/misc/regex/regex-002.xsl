<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test matches() function with line mode. See test bug 867 and spec bug 16809. -->
  <?spec fo#func-matches?>

  <xsl:template match="/">
    <out>;
      <true result="{matches(doc, '^Mary', 'm')}"/>;
      <true result="{matches(doc, '\n^Mary', 'm')}"/>;      
      <false result="{matches(doc, '^Mary', '')}"/>;      
      <true result="{matches(doc, '^And', 'm')}"/>;
      <false result="{matches(doc, '^And', '')}"/>;    
      <true result="{matches(doc, '^And.*went$', 'm')}"/>;   
      <false result="{matches(doc, '^And.*went$', '')}"/>;   
      <true result="{matches(doc, 'foot$', 'm')}"/>;   
      <true result="{matches(doc, 'foot\n$', '')}"/>; 
      <false result="{matches(doc, 'foot$', '')}"/>;   
      <true result="{matches(doc, '^\nMary.*foot\n$', 's')}"/>; 
      <false result="{matches(doc, '^\nMary.*foot$', '')}"/>;   
      <false result="{matches(doc, '\n^Mary.*foot$', 'm')}"/>;   
      <true result="{matches(doc, 'Mary.*foot', 's')}"/>;   
      <false result="{matches(doc, 'Mary.*foot', 'm')}"/>;   
    </out>
  </xsl:template>

</xsl:stylesheet>