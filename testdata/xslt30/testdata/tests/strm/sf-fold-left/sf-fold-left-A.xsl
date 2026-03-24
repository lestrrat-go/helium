<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="http://www.w3.org/local-functions"
    exclude-result-prefixes="f map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
    <xsl:function name="f:add" as="xs:double">
      <xsl:param name="x" as="xs:double"/>
      <xsl:param name="y" as="xs:double"/>
      <xsl:sequence select="$x + $y"/>
    </xsl:function>
    
    <!-- TODO: fold-left is defined in the current XSLT 3.0 draft to have the operand usage N (Navigation) making
     it non-streamable. This should be fixed; it should have type-determined usage based on the type of the first
     argument of the function provided in the third argument, if this is known. In the meantime, we explicitly 
     atomize the sequence to be processed -->

    
    <!-- Test of xsl:source-document with fold-left() -->
    
    <xsl:template name="s-003" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="fold-left(./BOOKLIST/BOOKS/ITEM/PRICE/data(), 0, f:add#2)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with fold-left(), filtered with a motionless predicate -->
    
    <xsl:template name="s-004" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="fold-left(./BOOKLIST/BOOKS/ITEM[@CAT='P']/PRICE/data(), 0, f:add#2)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with fold-left(), filtered with a positional predicate -->
    
    <xsl:template name="s-005" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="fold-left(./BOOKLIST/BOOKS/ITEM[position() lt 4]/PRICE/data(), 0, f:add#2)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with fold-left(), filtered with a positional predicate -->
    
    <xsl:template name="s-006" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="fold-left(./BOOKLIST/BOOKS/*[position() lt 4]/PRICE/data(), 0, f:add#2)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with fold-left(), filtered with a positional predicate -->
    
    <xsl:template name="s-007" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:sequence select="fold-left(./BOOKLIST/BOOKS/*:ITEM[position() lt 4]/PRICE/data(), 0, f:add#2)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with fold-left(), filtered using data() to make it streamable -->
    
    <xsl:template name="s-008" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="round(fold-left(./BOOKLIST/BOOKS/ITEM/PAGES/data()[. &lt; 1000][. &gt; 0], 0, f:add#2))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with fold-left(), filtered using text() to make it streamable -->
    
    <xsl:template name="s-009" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="round(fold-left(./BOOKLIST/BOOKS/ITEM/PAGES/text()[. &lt; 1000][. &gt; 0]/data(), 0, f:add#2))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with fold-left(), filtered using number() to make it streamable -->
    
    <xsl:template name="s-010" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="round(fold-left(./BOOKLIST/BOOKS/ITEM/PAGES/number()[. &lt; 1000][. &gt; 0], 0, f:add#2))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with fold-left(), selecting nothing and returning the $zero result -->
    
    <xsl:template name="s-011" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="fold-left(ITEM/PAGES/data(), 42, f:add#2)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of fold-left() selecting both streamed nodes and literals -->
    
    <xsl:template name="s-012" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="fold-left((./BOOKLIST/BOOKS/ITEM/PAGES/number(), 31, 32), 0, f:add#2)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of fold-left() selecting both streamed nodes and literals while also filtering -->
    
    <xsl:template name="s-013" use-when="true() or $RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="fold-left((tail(./BOOKLIST/BOOKS/ITEM/PAGES)/number(), 31, 32), 0, f:add#2)"/>
        </out>
      </xsl:source-document>
    </xsl:template>   
    
    <!-- Test of xsl:source-document with sum of a computed value -->
    
    <xsl:template name="s-015" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="format-number(fold-left(for $d in data(outermost(//DIMENSIONS))
                                                  return let $x := tokenize($d, '\s')!number() 
                                                  return $x[1]*$x[2]*$x[3], 0, f:add#2), '99.999')"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with sum of a computed value -->
    
    <xsl:template name="s-016" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="xs:integer(round(fold-left(account/transaction/(@value*2), 0, f:add#2)))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with fold-left(), filtered using copy-of() to make it streamable -->
    
    <xsl:template name="s-017" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="round(fold-left(./BOOKLIST/BOOKS/ITEM/PAGES/copy-of()[. &lt; 1000][. &gt; 0], 0, f:add#2))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with fold-left() applied to an attribute -->
    
    <xsl:template name="s-018" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="round(fold-left(account/transaction/@value/data(), 0, f:add#2))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with fold-left() over attribute values, filtered -->
    
    <xsl:template name="s-019" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="round(fold-left(account/transaction/@value[xs:decimal(.) gt 0]/data(), 0, f:add#2))"/>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- Test of xsl:source-document with fold-left() over attribute values, computed -->
    
    <xsl:template name="s-020" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="round(fold-left(account/transaction/abs(@value), 0, f:add#2))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- fold-left() of a conditional value -->
    
    <xsl:template name="s-021" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="round(fold-left(account/transaction/
                (if (xs:date(@date) lt xs:date('2020-01-01')) then +@value else (@value+1)), 0, f:add#2))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- fold-left() with a loop used to compute each item -->
    
    <xsl:template name="s-022" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="fold-left(BOOKLIST/BOOKS/ITEM/DIMENSIONS!xs:NMTOKENS(.)!xs:decimal(.), 0, f:add#2)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document calling fold-left() using //x/y -->
    
    <xsl:template name="s-023" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="fold-left(/*/*/ITEM/DIMENSIONS!xs:NMTOKENS(.)!xs:decimal(.), 0, f:add#2)"/>
        </out>
      </xsl:source-document>
    </xsl:template>                            
    
</xsl:stylesheet>